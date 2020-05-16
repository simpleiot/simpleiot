module Pages.Users exposing (Model, Msg, page)

import Components.Form as Form
import Data.User as U
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Generated.Params as Params
import Global
import Spa.Page
import Spa.Types as Types
import Utils.Spa exposing (Page)
import Utils.Styles exposing (palette, size)


page : Page Params.Users Model Msg model msg appMsg
page =
    Spa.Page.component
        { title = always "Users"
        , init = always init
        , update = update
        , subscriptions = subscriptions
        , view = view
        }



-- INIT


type alias Model =
    { userEdit : Maybe U.User
    }


init : Params.Users -> ( Model, Cmd Msg, Cmd Global.Msg )
init _ =
    ( { userEdit = Nothing
      }
    , Cmd.none
    , Spa.Page.send Global.RequestUsers
    )



-- UPDATE


type Msg
    = PostUser U.User
    | EditUser U.User
    | DiscardUserEdits
    | NewUser


update : Types.PageContext route Global.Model -> Msg -> Model -> ( Model, Cmd Msg, Cmd Global.Msg )
update context msg model =
    case msg of
        EditUser user ->
            ( { model | userEdit = Just user }
            , Cmd.none
            , Cmd.none
            )

        DiscardUserEdits ->
            ( { model | userEdit = Nothing }
            , Cmd.none
            , Cmd.none
            )

        PostUser user ->
            ( { model | userEdit = Nothing }
            , Cmd.none
            , case context.global of
                Global.SignedIn _ ->
                    Spa.Page.send <| Global.UpdateUser user

                Global.SignedOut _ ->
                    Cmd.none
            )

        NewUser ->
            ( { model | userEdit = Just U.empty }
            , Cmd.none
            , Cmd.none
            )



-- SUBSCRIPTIONS


subscriptions : Types.PageContext route Global.Model -> Model -> Sub Msg
subscriptions _ _ =
    Sub.none



-- VIEW


view : Types.PageContext route Global.Model -> Model -> Element Msg
view context model =
    case context.global of
        Global.SignedIn sess ->
            column
                [ width fill, spacing 32 ]
                [ el [ padding 16, Font.size 24 ] <| text "Users"
                , el [ padding 16, width fill, Font.bold ] <| Form.button "new user" palette.green NewUser
                , viewUsers sess.data.users model.userEdit
                ]

        _ ->
            el [ padding 16 ] <| text "Sign in to view users."


type alias UserMod =
    { user : U.User
    , mod : Bool
    }


mergeUserEdit : List U.User -> Maybe U.User -> List UserMod
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
                [ { user = edit, mod = True } ] ++ usersMapped

            else
                usersMapped

        Nothing ->
            List.map (\u -> { user = u, mod = False }) users


viewUsers : List U.User -> Maybe U.User -> Element Msg
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


viewUser : Bool -> U.User -> Element Msg
viewUser modded user =
    wrappedRow
        ([ width fill
         , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
         , Border.color palette.black
         , spacing 6
         ]
            ++ (if modded then
                    [ Background.color palette.orange
                    , below <|
                        Form.buttonRow
                            [ Form.button "discard" palette.pale <| DiscardUserEdits
                            , Form.button "save" palette.green <| PostUser user
                            ]
                    ]

                else
                    []
               )
        )
        [ Form.viewTextProperty
            { name = "First name"
            , value = user.first
            , action = \x -> EditUser { user | first = x }
            }
        , Form.viewTextProperty
            { name = "Last name"
            , value = user.last
            , action = \x -> EditUser { user | last = x }
            }
        , Form.viewTextProperty
            { name = "Email"
            , value = user.email
            , action = \x -> EditUser { user | email = x }
            }
        , Form.viewTextProperty
            { name = "Password"
            , value = user.pass
            , action = \x -> EditUser { user | pass = x }
            }
        ]
