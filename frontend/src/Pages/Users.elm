module Pages.Users exposing (Model, Msg, Params, page)

import Api.Auth exposing (Auth)
import Api.Data as Data exposing (Data)
import Api.Response exposing (Response)
import Api.User as User exposing (User)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Http
import Shared
import Spa.Document exposing (Document)
import Spa.Generated.Route as Route
import Spa.Page as Page exposing (Page)
import Spa.Url exposing (Url)
import UI.Form as Form
import UI.Icon as Icon
import UI.Style as Style
import Utils.Route


page : Page Params Model Msg
page =
    Page.application
        { init = init
        , update = update
        , subscriptions = subscriptions
        , view = view
        , save = save
        , load = load
        }



-- INIT


type alias Params =
    ()


type alias Model =
    { userEdit : Maybe User
    , users : List User
    , auth : Auth
    , error : Maybe String
    }


defaultModel : Model
defaultModel =
    Model
        Nothing
        []
        { email = "", token = "", isRoot = False }
        Nothing


init : Shared.Model -> Url Params -> ( Model, Cmd Msg )
init shared _ =
    case shared.auth of
        Just auth ->
            let
                model =
                    { defaultModel | auth = auth }
            in
            ( model
            , updateUsers model
            )

        Nothing ->
            ( defaultModel
            , Utils.Route.navigate shared.key Route.SignIn
            )



-- UPDATE


type Msg
    = New
    | Edit User
    | DiscardEdits
    | ApiUpdate User
    | ApiDelete String
    | ApiRespUpdate (Data Response)
    | ApiRespList (Data (List User))
    | ApiRespDelete (Data Response)


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        New ->
            ( { model | userEdit = Just User.empty }
            , Cmd.none
            )

        Edit user ->
            ( { model | userEdit = Just user }
            , Cmd.none
            )

        DiscardEdits ->
            ( { model | userEdit = Nothing }
            , Cmd.none
            )

        ApiUpdate user ->
            let
                -- optimistically update users
                users =
                    List.map
                        (\u ->
                            if u.id == user.id then
                                user

                            else
                                u
                        )
                        model.users
            in
            ( { model | userEdit = Nothing, users = users }
            , User.update
                { token = model.auth.token
                , user = user
                , onResponse = ApiRespUpdate
                }
            )

        ApiDelete id ->
            let
                -- optimisitically delete user
                users =
                    List.filter (\u -> u.id /= id) model.users
            in
            ( { model | users = users }
            , User.delete
                { token = model.auth.token
                , id = id
                , onResponse = ApiRespDelete
                }
            )

        ApiRespUpdate resp ->
            case resp of
                Data.Success _ ->
                    ( model, updateUsers model )

                Data.Failure err ->
                    ( popError "Error updating user" err model, updateUsers model )

                _ ->
                    ( model, updateUsers model )

        ApiRespList resp ->
            case resp of
                Data.Success users ->
                    ( { model | users = users }, Cmd.none )

                Data.Failure err ->
                    ( popError "Error getting users" err model, Cmd.none )

                _ ->
                    ( model, Cmd.none )

        ApiRespDelete resp ->
            case resp of
                Data.Success _ ->
                    ( model, updateUsers model )

                Data.Failure err ->
                    ( popError "Error deleting user" err model, updateUsers model )

                _ ->
                    ( model, updateUsers model )


popError : String -> Http.Error -> Model -> Model
popError desc err model =
    { model | error = Just (desc ++ ": " ++ Data.errorToString err) }


updateUsers : Model -> Cmd Msg
updateUsers model =
    User.list { onResponse = ApiRespList, token = model.auth.token }


save : Model -> Shared.Model -> Shared.Model
save model shared =
    { shared
        | error =
            case model.error of
                Nothing ->
                    shared.error

                Just _ ->
                    model.error
        , lastError =
            case model.error of
                Nothing ->
                    shared.lastError

                Just _ ->
                    shared.now
    }


load : Shared.Model -> Model -> ( Model, Cmd Msg )
load _ model =
    ( { model | error = Nothing }, Cmd.none )


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.none



-- VIEW


view : Model -> Document Msg
view model =
    { title = "SIOT Users"
    , body =
        [ column
            [ width fill, spacing 32 ]
            [ el Style.h2 <| text "Users"
            , el [ padding 16, width fill, Font.bold ] <|
                Form.button
                    { label = "new user"
                    , color = Style.colors.blue
                    , onPress = New
                    }
            , viewUsers model.users model.userEdit
            ]
        ]
    }


viewUsers : List User -> Maybe User -> Element Msg
viewUsers users userEdit =
    column
        [ width fill
        , spacing 40
        ]
    <|
        List.map
            (\user ->
                viewUser user.mod user.user
            )
        <|
            mergeUserEdit users userEdit


type alias UserMod =
    { user : User
    , mod : Bool
    }


mergeUserEdit : List User -> Maybe User -> List UserMod
mergeUserEdit users userEdit =
    case userEdit of
        Just edit ->
            let
                usersMapped =
                    List.map
                        (\u ->
                            if edit.id == u.id then
                                { user = edit, mod = True }

                            else
                                { user = u, mod = False }
                        )
                        users
            in
            if edit.id == "" then
                { user = edit, mod = True } :: usersMapped

            else
                usersMapped

        Nothing ->
            List.map (\u -> { user = u, mod = False }) users


viewUser : Bool -> User -> Element Msg
viewUser modded user =
    wrappedRow
        ([ width fill
         , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
         , Border.color Style.colors.black
         , spacing 6
         ]
            ++ (if modded then
                    [ Background.color Style.colors.orange
                    , below <|
                        Form.buttonRow
                            [ Form.button
                                { label = "discard"
                                , color = Style.colors.gray
                                , onPress = DiscardEdits
                                }
                            , Form.button
                                { label = "save"
                                , color = Style.colors.blue
                                , onPress = ApiUpdate user
                                }
                            ]
                    ]

                else
                    []
               )
        )
        [ Form.viewTextProperty
            { name = "First name"
            , value = user.first
            , action = \x -> Edit { user | first = x }
            }
        , Form.viewTextProperty
            { name = "Last name"
            , value = user.last
            , action = \x -> Edit { user | last = x }
            }
        , Form.viewTextProperty
            { name = "Phone #"
            , value = user.phone
            , action = \x -> Edit { user | phone = x }
            }
        , Form.viewTextProperty
            { name = "Email"
            , value = user.email
            , action = \x -> Edit { user | email = x }
            }
        , Form.viewTextProperty
            { name = "Password"
            , value = user.pass
            , action = \x -> Edit { user | pass = x }
            }
        , Icon.userX (ApiDelete user.id)
        ]
