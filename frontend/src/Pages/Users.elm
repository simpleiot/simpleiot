module Pages.Users exposing (Flags, Model, Msg, page)

import Data.User as U
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Global
import Page exposing (Document, Page)
import UI.Form as Form
import UI.Icon as Icon
import UI.Style as Style


type alias Flags =
    ()


type alias Model =
    { userEdit : Maybe U.User
    }


type Msg
    = PostUser U.User
    | EditUser U.User
    | DiscardUserEdits
    | DeleteUser String
    | NewUser


page : Page Flags Model Msg
page =
    Page.component
        { init = init
        , update = update
        , subscriptions = subscriptions
        , view = view
        }


init : Global.Model -> Flags -> ( Model, Cmd Msg, Cmd Global.Msg )
init _ _ =
    ( Model Nothing, Cmd.none, Global.send Global.RequestUsers )


update : Global.Model -> Msg -> Model -> ( Model, Cmd Msg, Cmd Global.Msg )
update global msg model =
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
            , case global.auth of
                Global.SignedIn _ ->
                    Global.send <| Global.UpdateUser user

                Global.SignedOut _ ->
                    Cmd.none
            )

        DeleteUser id ->
            ( model
            , Cmd.none
            , Global.send <| Global.DeleteUser id
            )

        NewUser ->
            ( { model | userEdit = Just U.empty }
            , Cmd.none
            , Cmd.none
            )


subscriptions : Global.Model -> Model -> Sub Msg
subscriptions _ _ =
    Sub.none


view : Global.Model -> Model -> Document Msg
view global model =
    { title = "Users"
    , body =
        [ case global.auth of
            Global.SignedIn sess ->
                column
                    [ width fill, spacing 32 ]
                    [ el Style.h2 <| text "Users"
                    , el [ padding 16, width fill, Font.bold ] <|
                        Form.button "new user" Style.colors.blue NewUser
                    , viewUsers sess.data.users model.userEdit
                    ]

            _ ->
                el [ padding 16 ] <| text "Sign in to view users."
        ]
    }


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
                { user = edit, mod = True } :: usersMapped

            else
                usersMapped

        Nothing ->
            List.map (\u -> { user = u, mod = False }) users


viewUser : Bool -> U.User -> Element Msg
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
                            [ Form.button "discard"
                                Style.colors.gray
                              <|
                                DiscardUserEdits
                            , Form.button "save"
                                Style.colors.blue
                              <|
                                PostUser user
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
        , Icon.userX (DeleteUser user.id)
        ]
